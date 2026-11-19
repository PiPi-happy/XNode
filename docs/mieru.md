# Mieru support

XNode runs Mieru through the Mieru author's [MBox fork of sing-box](https://github.com/enfein/mbox), which registers both Mieru inbound and outbound implementations. XNode selects this sing-box backend for a panel node whose protocol is `mieru`, even when the node's configured default kernel is Xray.

No standalone Mita service is needed. The Mieru server runs inside XNode's sing-box instance and shares its lifecycle, user updates, traffic tracker, and listener handling.

## Panel mapping

- The panel's Mieru `transport` is passed into the MBox inbound (`TCP` or `UDP`).
- The panel's server port is used for the inbound listener.
- Each XBoard user UUID is used as both Mieru username and password, matching XBoard's Mieru profile generation.
- `traffic_pattern` is passed unchanged from the panel's Base64 value to the MBox inbound.

The MBox fork is pinned in `go.mod` to the author's `v1.14.3` release so builds use a reproducible Mieru-capable sing-box implementation. The fork warns that it does not guarantee every sing-box feature behaves identically to upstream; Mieru support is the feature XNode relies on here.
