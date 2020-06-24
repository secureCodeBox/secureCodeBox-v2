import * as k8s from "@kubernetes/client-node";

// configure k8s client
const kc = new k8s.KubeConfig();
kc.loadFromDefault();

const k8sApiCRD = kc.makeApiClient(k8s.CustomObjectsApi);

async function startSubsequentSecureCodeBoxScan({
  name,
  parentScan,
  scanType,
  parameters,
  generatedBy,
}) {
  const scanDefinition = {
    apiVersion: "execution.experimental.securecodebox.io/v1",
    kind: "Scan",
    metadata: {
      generateName: `${name}-`,
      labels: {
        ...parentScan.metadata.labels,
      },
      annotations: {
        "securecodebox.io/hook": "declarative-subsequent-scans",
        "securecodebox.io/parent-scan": parentScan.metadata.name,
        "cascading.securecodebox.io/generated-by": generatedBy,
      },
      ownerReferences: [
        {
          apiVersion: "execution.experimental.securecodebox.io/v1",
          blockOwnerDeletion: true,
          controller: true,
          kind: "Scan",
          name: parentScan.metadata.name,
          uid: parentScan.metadata.uid,
        },
      ],
    },
    spec: {
      scanType,
      parameters,
    },
  };

  try {
    // Starting another subsequent sslyze scan based on the nmap results
    // found at: https://github.com/kubernetes-client/javascript/blob/79736b9a608c18d818de61a6b44503a08ea3a78f/src/gen/api/customObjectsApi.ts#L209
    await k8sApiCRD.createNamespacedCustomObject(
      "execution.experimental.securecodebox.io",
      "v1",
      "default",
      "scans",
      scanDefinition,
      "false"
    );
  } catch (error) {
    console.error(`Failed to start Scan ${name}`);
    console.error(error);
  }
}

module.exports.startSubsequentSecureCodeBoxScan = startSubsequentSecureCodeBoxScan;

async function getCascadingRulesFromCluster() {
  try {
    const namespace = process.env["NAMESPACE"];
    const { body } = await k8sApiCRD.listNamespacedCustomObject(
      "cascading.experimental.securecodebox.io",
      "v1",
      namespace,
      "cascadingrules"
    );
    console.log("got CascadingRules");
    console.log(body);
    console.log(JSON.stringify(body));
    return body.items;
  } catch (err) {
    console.error("Failed to get CascadingRules from the kubernetes api");
    console.error(err);
    process.exit(1);
  }
}
module.exports.getCascadingRulesFromCluster = getCascadingRulesFromCluster;

enum LabelSelectorRequirementOperator {
  In,
  NotIn,
  Exists,
  DoesNotExist,
}

// See: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.18/#labelselectorrequirement-v1-meta
// Re created in TS because the included types suck 😕
interface LabelSelectorRequirement {
  key: string;
  values: string;

  operator: LabelSelectorRequirementOperator;
}

function generateLabelSelectorString(
  matchExpression: Array<LabelSelectorRequirement>,
  matchLabels: Map<string, string>
): string {
  // Convert matchLabels to matchExpression syntax
  matchExpression;
  return "";
}