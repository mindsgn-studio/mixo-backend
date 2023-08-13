"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.trpc = exports.createContext = void 0;
var server_1 = require("@trpc/server");
var createContext = function (_a) {
    var req = _a.req, res = _a.res;
    return ({
        req: req,
        res: res,
    });
};
exports.createContext = createContext;
exports.trpc = server_1.initTRPC.context().create();
