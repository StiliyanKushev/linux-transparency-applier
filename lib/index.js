const path = require('path')
const cp = require('child_process')
const os = require('os')
const fs = require('fs')
const ffi = require('ffi-napi')

// Define GoLang exported functions here
const Functions = ffi.Library(path.join(__dirname, './lta.so'), {
    'CreateSandboxParentWindow'  :    [ 'int',    [ 'int', 'int', 'int', 'int' ] ],
    'BindXNestedToWindow'        :    [ 'int',    [ 'int' ] ],
    'GetWindowIdsByDisplayId'    :    [ 'string', [ 'int', 'int' ] ],
    'TransformWindow'            :    [ 'void',   [ 'int', 'int' ] ],
    'LinkEventsWithChild'        :    [ 'void',   [ 'int', 'int', 'string', 'string' ] ],
    'CreateExcluderShape'        :    [ 'void',   [ 'string', 'int', 'int' ] ],
    'ResetWindowShape'           :    [ 'void',   [ 'int', 'int' ] ]
})

/**
 * @name MakeTransparentEnv
 * @param {Number} x Default is 0
 * @param {Number} y Default is 0
 * @param {Number} width Default is 800
 * @param {Number} height Default is 600
 * @description This function creates a special drawable window with nested x instance inside.
 * @returns { ParentWindowId: Number, DisplayId: Number }
 */
function MakeTransparentEnv(x = 0, y = 0, width = 800, height = 600) {
    // create parent window
    const ParentWindowId = Functions.CreateSandboxParentWindow(x, y, width, height)

    if(ParentWindowId == 0) {
        console.log('Parent window creation failed')
        return false
    }
    
    // setup xnest for that window
    const DisplayId = Functions.BindXNestedToWindow(ParentWindowId)

    if(DisplayId == 0) {
        console.log('Display creation failed')
        return false
    }

    return { ParentWindowId, DisplayId }
}

/**
 * @name ApplyTransparentFilter
 * @param {String} hexColor Trancparency color in hex format
 * @param {Number} windowId Window id of the parent window
 * @param {Number?} displayId Display id of the parent window (default is 0)
 * @description This function applies a transparent filter to the window
 */
function ApplyTransparentFilter(hexColor, windowId, displayId = 0) {
    Functions.CreateExcluderShape(hexColor, windowId, displayId)
}

/**
 * @name ResetWindowShape
 * @param {Number} windowId Window id of the parent window
 * @param {Number?} displayId Display id of the parent window (default is 0)
 * @description This function resets the window shape to default
 */
function ResetWindowShape(windowId, displayId = 0) {
    Functions.ResetWindowShape(windowId, displayId)
}

/**
 * @name RunProgramInTransparentEnv
 * @param {String} launchCmd 
 * @param {Number} displayId 
 * @param {Number} parentWindowId 
 * @param {String} transparencyColor 
 * @description runs the program inside the nested x instance and links the events
 * @returns {Number[]} Window ids of the running windows inside xnest after launch cmd
 */
function RunProgramInTransparentEnv(launchCmd, displayId, parentWindowId, transparencyColor) {
    const programName = Math.random().toString().replace('.', '')

    try {
        // make a new bash script that acts as this program but we run the bash script with some args
        // that let any app be ran into an x org server
        const patched = path.join(os.tmpdir(), `./${programName}.${displayId}`)
        fs.writeFileSync(patched, `#!/bin/sh\n${launchCmd} --name=${programName}\n`)

        // set the display variable and execute the script
        let patchedProgram = `DISPLAY=:${displayId} sudo sh ${patched} ` + [
            `--enable-greasemonkey`, 
            `--enable-user-scripts`, 
            `--enable-extensions`, 
            `"$@"`
        ].join(' ')
        cp.exec(patchedProgram)

        // now let's store all initial windows as the ones we need to keep
        const windowIdsString = Functions.GetWindowIdsByDisplayId(displayId, 500)
        const windowIdsArray = windowIdsString.split(',')

        // since we're not sure how many windows there are spawned by default
        // we loop all of them and move them top left and resize them to the window
        for(let foundWindowId of windowIdsArray) {
            Functions.TransformWindow(foundWindowId, displayId)
        }

        // we also link each child window to the parent window
        // that means resize events are linked, close events are linked
        // minimize, maximize events are linked, and so on and so fort
        Functions.LinkEventsWithChild(parentWindowId, displayId, windowIdsString, transparencyColor)

        return windowIdsArray
    }
    catch (error) {
        console.log(error)
        return 0
    }
}

module.exports = {
    ResetWindowShape,
    MakeTransparentEnv,
    ApplyTransparentFilter,
    RunProgramInTransparentEnv
}