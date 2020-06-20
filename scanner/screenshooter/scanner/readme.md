This scanner is using [gowitness](https://github.com/sensepost/gowitness).

This works pretty well, but has currently two technical problem:

- The cli command to take a single screenshot is missing a option to change the automatically generated filename. There is a PR open to add a option to override this. The image used atm is build from this PR. https://github.com/sensepost/gowitness/pull/53
- The screenshot taken by chrome only has read permission to the chrome / gowitness linux user. The lurcher cannot acess it. This was monkey patched by adding a wrapper.sh script which chmod's the screenshot file before terminating.
