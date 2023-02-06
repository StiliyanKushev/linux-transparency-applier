function assert(Description, Expression, NestedAsserts, ExtraLog) {

    if(NestedAsserts) {
        console.log(`-- ${Description}${ExtraLog ? ` -> ${ExtraLog}` : ''}`)
        return Expression()
    }

    function test() {
        try {
            const Result = Expression()
            return { Success: Result ? true : false, Result: Result }
        } catch (error) {
            return { Success: false, Result: error }
        }
    }

    const { Success, Result } = test()
    
    if(Success) {
        console.log(`✓ [OK] ${Description} - ${Result}${ExtraLog ? ` -> ${ExtraLog}` : ''}`)
    }
    else {
        console.log(`x [ER] ${Description} - ${Result}${ExtraLog ? ` -> ${ExtraLog}` : ''}`)
    }
}

module.exports = {
    assert,
}