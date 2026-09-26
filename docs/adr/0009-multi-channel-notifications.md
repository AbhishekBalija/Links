# Use Multi-Channel Notifications

Decision: Use in-app notifications as the permanent record, email for important official fallback, and web push for opted-in users when the app is closed.

Reason:

- Students may not keep LINKS open.
- Placement links and shortlisting updates are time-sensitive.
- Web push depends on browser/device permission and is not guaranteed.
- Email is more reliable for official communication.

Trade-off:

- Requires notification preferences, delivery tracking, and a worker/outbox system.
