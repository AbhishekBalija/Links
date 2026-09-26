# Use Modular Monolith First

Decision: Use one Go REST API with internal modules.

Reason:

- The domain is connected.
- Transactions matter.
- Deployment stays simple.
- The team can move faster.

Trade-off:

- Requires discipline to maintain module boundaries.
