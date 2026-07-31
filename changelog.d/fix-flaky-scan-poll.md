<!-- file: changelog.d/fix-flaky-scan-poll.md -->
<!-- version: 1.0.0 -->
<!-- guid: 6a91c73e-25b8-4f04-8d16-e3b7025fa9c1 -->
<!-- last-edited: 2026-07-30 -->

### Fixed

#### Flaky scan-handler test under `-tags sqlite`

`TestScanHandlers` polled for one second (ten 100ms ticks) and then declared
"scan did not finish". The scan contacts subtitle providers over the network,
so that ceiling was always tight; it tipped over once provider responses began
being validated, because a stub answering `200` with junk no longer counts as a
hit and the scan works through more providers before concluding.

The symptom was misleading: it passed in isolation and failed in a full package
run, which reads as cross-test pollution rather than a scan that needs more
than a second. The poll now runs to a 30s deadline and still exits as soon as
the scan reports itself finished, so the normal case is no slower.
