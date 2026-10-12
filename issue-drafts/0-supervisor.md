Migrate the `stock` CLI from Finnhub to the Massive (formerly Polygon.io) market data API, and modernise the codebase along the way. The work covers Go modules and CI, a testable provider abstraction, bug fixes, and env-based API key handling.

The children run in order. Each one builds on the previous one's merged code.

```changathi-supervisor
version: 1
children:
  - issue: 1
    depends_on: []
  - issue: 2
    depends_on: [1]
  - issue: 3
    depends_on: [2]
  - issue: 4
    depends_on: [3]
```
