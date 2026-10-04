# Pi qualification tooling and records

This directory holds test and qualification material, separate from published
user documentation. The installed-runtime scripts remain alongside it at
`../pi-native-resources.mjs` and `../pi-native-rpc.py`.

- `evidence/`: case contract, templates, example vectors and historical candidate records.
- `reference/`: archived implementation reviews and deployment-validation observations.

The offline checker reads the relocated case contract and the Python tests read
the relocated templates and examples. Historical candidate records retain their
original source paths, revisions and hashes: moving these records does not
requalify them or make their historical paths describe the current checkout.
The candidate records are not current runtime configuration or new test results.
The historical documents likewise retain their original paths in recorded claims.

Current usage instructions are in `docs/`. For the record format and example
vectors, see `evidence/README.md`; for the observed deployment checks, see
`reference/pi-native-validation.md`.

Run the offline checks from the repository root:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s scripts/tests -p 'test_pi_*.py'
```
