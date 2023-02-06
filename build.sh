# build the go library
CGO_LDFLAGS="-Wl,--allow-multiple-definition" go build -o ./lib/lta.so -buildmode=c-shared ./src/main.go &&

# pack the npm module
cd ./lib && npm pack --pack-destination="../out" && cd .. &&

# run nodejs lib tests
cd ./test && npm i ../out/lta-1.0.0.tgz && node ./index.js