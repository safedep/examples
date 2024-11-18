var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-node";
import { InsightService } from "@buf/safedep_api.connectrpc_es/safedep/services/insights/v2/insights_connect";
import { Ecosystem } from "@buf/safedep_api.bufbuild_es/safedep/messages/package/v1/ecosystem_pb";
function main() {
    return __awaiter(this, void 0, void 0, function* () {
        const transport = createConnectTransport({
            baseUrl: "https://api.safedep.io",
            httpVersion: "1.1",
        });
        const client = createPromiseClient(InsightService, transport);
        const res = yield client.getPackageVersionInsight({
            packageVersion: {
                package: {
                    ecosystem: Ecosystem.NPM,
                    name: "lodash",
                },
                version: "4.17.21",
            }
        });
        console.log(res.toJson());
    });
}
main();
