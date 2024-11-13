An example to show distributed task processing in golang.
```
                                   ┌─────────────┐
                                   │             │
                     JobChannel    │   Worker    │    ResultPool
Jobs ─────────────────────────────►│     ID=1    ├───────────────► Results
                                   │             │
                                   └─────────────┘
                                         │
                                         │ WorkerPool
                                         ▼
                                   Available Workers
```