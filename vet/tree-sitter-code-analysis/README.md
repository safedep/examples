# Call Graph

## Usage

Build the tools:

```shell
make
```

Run call graph generator on a Python script:

```shell
./bin/cg samples/4.py
```

Optionally, use `tree-sitter` to visualize the CST:

```shell
npm install
```

```shell
npx tree-sitter parse samples/4.py
```

## Reference

* https://arxiv.org/pdf/2103.00587
