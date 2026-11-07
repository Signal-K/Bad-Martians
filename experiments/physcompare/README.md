# physcompare

Experiment behind `docs/PHYSICS.md`: the same rover and terrain scenarios run
against our own solver and two off-the-shelf ones (Box2D port, Chipmunk).

    go run . results.json          # prints markdown tables, writes raw JSON
    GOOS=js GOARCH=wasm go build -o pc.wasm . && node wasm_exec_node.js pc.wasm   # wasm run (needs a small environment: env -i)

It is a separate Go module so Box2D and Chipmunk never reach the game build.
`sizecheck/` holds tiny programs used to measure each library's WASM size.
