# Wetware Test Plans

This directory contains test-plans for use with [Testground](https://github.com/testground/testground).

These are cluster-level integration tests for the distributed protocols used by wetware.

## Dependencies

- [Testground](https://github.com/testground/testground)
- [Docker](https://www.docker.com/)

## Usage

### Initial setup

Estimated time:  about 5 minutes

1. Run `testground daemon`.  This will create a `$TESTGROUND_HOME` directory if it does not already exist (by default `$HOME/testground`)
2. Link this directory to your testground home directory with `ln -s <wetware root>/test-plans <testground home>/plans/ww`.  Make sure you replace the root paths match those on your system.
3. Run `testground plan list` and check that the `ww` plan appears.

### Running test plans

Estimated time:  about 5 minutes

A test plan is a set of test cases.  To view the test cases for the `ww` plan:

```bash
testground describe --plan ww
```

To run the `announce` test case, run:

```bash
testground run single --plan ww --testcase announce --builder docker:go --runner local:docker --instances=2
```

To vary the number of peers in the test cluster, change `--instances=2`.  N.B.: most tests require at least `--instances=2`.  See `manifest.toml`.

The test-case can be varied by changing the argument to the `--plan` flag.
