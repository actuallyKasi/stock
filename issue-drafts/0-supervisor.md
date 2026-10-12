Migrate the `stock` CLI from Finnhub to the Massive (formerly Polygon.io) market data API, and modernise the codebase along the way. The work covers Go modules and CI, a testable provider abstraction, bug fixes, and env-based API key handling.

The children run in order. Each one builds on the previous one's merged code.

```changathi-supervisor
version: 1
children:
  - issue: A
    depends_on: []
  - issue: B
    depends_on: [A]
  - issue: C
    depends_on: [B]
  - issue: D
    depends_on: [C]
```
