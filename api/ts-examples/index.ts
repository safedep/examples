import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-node";
import { InsightService } from "@buf/safedep_api.connectrpc_es/safedep/services/insights/v2/insights_connect";
import { Ecosystem } from "@buf/safedep_api.bufbuild_es/safedep/messages/package/v1/ecosystem_pb";

async function main() {
  const transport = createConnectTransport({
    baseUrl: "https://api.safedep.io",
    httpVersion: "1.1",
  });

  const client = createPromiseClient(InsightService, transport);
  const res = await client.getPackageVersionInsight({
    packageVersion: {
      package: {
        ecosystem: Ecosystem.NPM,
        name: "lodash",
      },
      version: "4.17.21",
    }
  })

  console.log(res.toJson())
}

main()
