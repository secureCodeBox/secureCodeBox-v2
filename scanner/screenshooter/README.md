---
title: "Screenshooter"
path: "scanner/screenshooter"
category: "scanner"
usecase: "Website Screnshooter"
---

![gowitness logo](https://raw.githubusercontent.com/sensepost/gowitness/master/images/gowitness-logo.png)

Gowitness is a tool to take screenshots of websites. This integration lets you automate screenshot creation to get a quick(er) overview of a large number of web services.

This folder is named screenshooter as we might change the tool later. See [list of current problems / workarounds](./scanner/readme.md)

<!-- end -->

## Deployment

The scanType can be deployed via helm.

```bash
helm upgrade --install screenshooter ./scanner/screenshooter/
```

## Examples

A set of examples can be found in the [examples](./examples) folder.

- Example _secureCodeBox.io_ [scan](./examples/secureCodeBox.io/scan.yaml) and [findings](./examples/secureCodeBox.io/findings.yaml)
- Example _example.com_ [scan](./examples/secureCodeBox.io/scan.yaml) and [findings](./examples/secureCodeBox.io/findings.yaml)

## Configuration

To see how to configure gowitness see the example above.
More details on the option provided by gowitness can be found at the [gowitness github repository](https://github.com/sensepost/gowitness#usage-examples)

## Development

### Local setup

1. Clone the repository `git clone git@github.com:secureCodeBox/secureCodeBox-v2-alpha.git`
2. Ensure you have node.js installed
   - On MacOs with brew package manager: `brew install node`

### Parser Development

1. Install the dependencies `npm install`
2. Update the parser function here: `./parser/parser.js`
3. Update the parser tests here: `./parser/parser.test.js`
4. Run the testsuite: `npm test`
