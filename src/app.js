"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var express_1 = require("express");
var express_2 = require("@trpc/server/adapters/express");
var dotenv_1 = require("dotenv");
var cors_1 = require("cors");
var trpc_1 = require("./utils/trpc");
var database_1 = require("./utils/database");
var api_route_1 = require("./routes/api.route");
var trpc_panel_1 = require("trpc-panel");
dotenv_1.default.config();
var app = (0, express_1.default)();
app.use((0, cors_1.default)());
app.use('/api', (0, express_2.createExpressMiddleware)({
    router: api_route_1.apiRoute,
    createContext: trpc_1.createContext,
}));
app.use('/panel', function (_, res) {
    return res.send((0, trpc_panel_1.renderTrpcPanel)(api_route_1.apiRoute, { url: 'http://localhost:8080/api' }));
});
app.listen(process.env.PORT, function () {
    console.log("\uD83D\uDE80 Server listening on port ".concat(process.env.PORT));
    (0, database_1.database)();
});
