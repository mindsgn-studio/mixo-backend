"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (g && (g = 0, op[0] && (_ = 0)), _) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.apiRoute = exports.isAuthorizedProcedure = void 0;
var trpc_1 = require("../utils/trpc");
var zod_1 = require("zod");
var server_1 = require("@trpc/server");
var song_controller_1 = require("../controllers/song.controller");
var isAuthorized = trpc_1.trpc.middleware(function (_a) {
    var ctx = _a.ctx, next = _a.next;
    if (!ctx) {
        throw new server_1.TRPCError({
            code: 'UNAUTHORIZED',
            message: 'You must be logged in to access this resource',
        });
    }
    return next();
});
exports.isAuthorizedProcedure = trpc_1.trpc.procedure.use(isAuthorized);
exports.apiRoute = trpc_1.trpc.router({
    getSongs: exports.isAuthorizedProcedure
        .input(zod_1.z.object({
        limit: zod_1.z.number().min(1).max(100),
        page: zod_1.z.number().min(1),
    }))
        .query(function (opts) { return __awaiter(void 0, void 0, void 0, function () {
        var input, limit, page, response;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    input = opts.input;
                    limit = input.limit, page = input.page;
                    if (!limit || !page) {
                        throw new server_1.TRPCError({
                            code: 'BAD_REQUEST',
                            message: 'expects limit and page as input',
                        });
                    }
                    return [4 /*yield*/, (0, song_controller_1.getAllSongs)(input)];
                case 1:
                    response = _a.sent();
                    return [2 /*return*/, response];
            }
        });
    }); }),
    searchSongs: exports.isAuthorizedProcedure
        .input(zod_1.z.object({
        searchText: zod_1.z.string(),
        limit: zod_1.z.number(),
        page: zod_1.z.number(),
    }))
        .query(function (opts) { return __awaiter(void 0, void 0, void 0, function () {
        var input, limit, page, searchText, response;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    input = opts.input;
                    limit = input.limit, page = input.page, searchText = input.searchText;
                    if (!limit || !page || !searchText) {
                        throw new server_1.TRPCError({
                            code: 'BAD_REQUEST',
                            message: 'expects limit and page as input',
                        });
                    }
                    return [4 /*yield*/, (0, song_controller_1.searchSongs)(input)];
                case 1:
                    response = _a.sent();
                    return [2 /*return*/, response];
            }
        });
    }); }),
    likeSong: exports.isAuthorizedProcedure
        .input(zod_1.z.object({
        id: zod_1.z.string(),
        userID: zod_1.z.string(),
    }))
        .mutation(function (opts) { return __awaiter(void 0, void 0, void 0, function () {
        var input, id, userID, response;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    input = opts.input;
                    id = input.id, userID = input.userID;
                    if (!id || !userID) {
                        throw new server_1.TRPCError({
                            code: 'BAD_REQUEST',
                            message: 'expects id as input',
                        });
                    }
                    return [4 /*yield*/, (0, song_controller_1.likeSong)(input)];
                case 1:
                    response = _a.sent();
                    return [2 /*return*/, response];
            }
        });
    }); }),
    getUserLikes: exports.isAuthorizedProcedure
        .input(zod_1.z.object({
        id: zod_1.z.string(),
        userID: zod_1.z.string(),
    }))
        .query(function (opts) { return __awaiter(void 0, void 0, void 0, function () {
        var input, id, userID, response;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    input = opts.input;
                    id = input.id, userID = input.userID;
                    if (!id || !userID) {
                        throw new server_1.TRPCError({
                            code: 'BAD_REQUEST',
                            message: 'expects id as input',
                        });
                    }
                    return [4 /*yield*/, (0, song_controller_1.likeSong)(input)];
                case 1:
                    response = _a.sent();
                    return [2 /*return*/, response];
            }
        });
    }); }),
    getSongLikes: exports.isAuthorizedProcedure
        .input(zod_1.z.object({
        id: zod_1.z.string(),
        userID: zod_1.z.string(),
    }))
        .query(function (opts) { return __awaiter(void 0, void 0, void 0, function () {
        var input, id, userID, response;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    input = opts.input;
                    id = input.id, userID = input.userID;
                    if (!id || !userID) {
                        throw new server_1.TRPCError({
                            code: 'BAD_REQUEST',
                            message: 'expects id as input',
                        });
                    }
                    return [4 /*yield*/, (0, song_controller_1.likeSong)(input)];
                case 1:
                    response = _a.sent();
                    return [2 /*return*/, response];
            }
        });
    }); }),
});
