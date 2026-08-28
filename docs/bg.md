# Background

I used @GOOGLE_AUTHENTICATOR@ app on iOS for a long time. At some point in
past there was speculation that google might discontinue it. I looked
around for an open source alternative, reviewed the code, and moved to
@ENTE_AUTH@. I do not store my authentication data in the cloud. Instead,
I periodically export an encrypted backup and keep copies on several
systems.

After getting a new iPhone (iOS v26.6), @ENTE_AUTH@ stopped
working and continuously displayed a spinner forever. Updating the app did not help, and
importing my encrypted backup failed too. I had backups, but no independent
way to restore them.

**This CLI was created to make sure that I will never be dependent on a
single mobile app, or be in trouble if I lose my phone.**

It follows Ente's documented export format in @ENTE_AUTH_EXPORT_DOC@. I am
also the author of @LIBSODIUM_JNA@, so the underlying crypto methods were
familiar.

**Update:** After deleting and re-installing the @ENTE_AUTH@ app, it was able
to import the encrypted JSON file from the old phone.
