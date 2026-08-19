# Test strategy

Tests should not require a live GCP project unless an integration suite is explicitly introduced.

Priorities:

- model validation
- analyzer classification and savings calculations
- mocked GCP service interfaces
- UI state transitions
- integration tests for live GCP APIs in a separate opt-in suite
