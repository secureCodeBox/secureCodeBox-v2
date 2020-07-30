const arg = require("arg");

async function parse(_, { scan }) {
  // parse the url screenshotted by passing the scans parameters and extracting the "--url param"
  const args = arg(
    {
      "--url": String,
      "-u": "--url",
    },
    { permissive: true, argv: scan.spec.parameters }
  );
  const websiteUrl = args["--url"];

  const downloadLink = scan.status.rawResultDownloadLink;

  return [
    {
      name: `Screenshot for ${websiteUrl}`,
      description: `Took a Screenshot for website: '${websiteUrl}'`,
      category: "Screenshot",
      location: websiteUrl,
      osi_layer: "APPLICATION",
      severity: "INFORMATIONAL",
      attributes: {
        downloadLink,
      },
    },
  ];
}

module.exports.parse = parse;
