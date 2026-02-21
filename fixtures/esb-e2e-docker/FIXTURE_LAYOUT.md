# fixture layout

Each template group is fully self-contained under its own directory.

- `core/template.yaml`
  - `core/functions/python/*`
  - `core/layers/common/*`
- `image-java-python/template.yaml`
  - `image-java-python/functions/java/*`
  - `image-java-python/images/lambda/*`
- `stateful/template.yaml`
  - `stateful/functions/python/*`
  - `stateful/layers/common/*`

All `CodeUri` / `ContentUri` in these templates resolve only within each subdirectory.

Repro script: `fixtures/esb-e2e-docker/reproduce-e2e-docker.sh`
