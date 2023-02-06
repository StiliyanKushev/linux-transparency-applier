const { 
    MakeTransparentEnv, 
    ApplyTransparentFilter,
    RunProgramInTransparentEnv 
} = require('lta')
const { assert } = require('./utils.js')

// keep everything running
process.stdin.on('data', () => {})

const PageScript = '(' + function () {
    window.onload = () => {
        // we make the page our specific color
        document.body.style.background = 'rgb(9, 255, 0)'
        
        // let's also listen events and notify our script here
        const ws = new WebSocket('ws://localhost:9898')

        ws.onopen = () => {
                                                ws.send('load')
            window.onresize             = () => ws.send('resize')
            document.oncontextmenu      = () => ws.send('context')
            document.onclose            = () => ws.send('close')
            document.onfullscreenchange = () => ws.send('fullscreen')
        }

    }
} + ')();'

// we run the ws server for later
const WebSocketServer = require('ws').Server
let MessageCallback = () => console.log('[WARNING] Callback not loaded')
new WebSocketServer({ port: 9898 }).on('connection', ws => {
    ws.on('message', message => {
        console.log(`[LOG] received: ${message}`)
        MessageCallback()
    })
})

// make sure all functions are exported correctly
assert('MakeTransparentEnv should exist', () => !!MakeTransparentEnv)
assert('ApplyTransparentFilter should exist', () => !!ApplyTransparentFilter)
assert('RunProgramInTransparentEnv should exist', () => !!RunProgramInTransparentEnv)

// Test window and display creation
assert('Parent window & display creation', () => {
    const { ParentWindowId, DisplayId } = MakeTransparentEnv(0, 0, 1200, 900)
    assert('window id', () => !!ParentWindowId && ParentWindowId > 0, null, ParentWindowId)
    assert('display id', () => !!DisplayId && DisplayId > 0, null, DisplayId)

    if(!ParentWindowId || !DisplayId) return

    // Test program running in nested x environment
    assert('window id inside nested environment', () => {
        return RunProgramInTransparentEnv(
            `google-chrome-stable --no-sandbox --new-window "data:text/html,<script>${PageScript}</script>"`, 
            DisplayId, 
            ParentWindowId,
        )
    })

    // we are ready to handle requests
    MessageCallback = () => ApplyTransparentFilter('#09ff00', ParentWindowId)
}, true)