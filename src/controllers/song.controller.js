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
exports.getUserLikes = exports.likeSong = exports.searchSongs = exports.getAllSongs = void 0;
var server_1 = require("@trpc/server");
var dotenv_1 = require("dotenv");
var song_schema_1 = require("../schema/song.schema");
var like_schema_1 = require("../schema/like.schema");
dotenv_1.default.config();
var getAllSongs = function (_a) {
    var limit = _a.limit, page = _a.page;
    return __awaiter(void 0, void 0, void 0, function () {
        var startIndex, totalQueryResult, total, totalPages, response, data, error_1;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    _b.trys.push([0, 3, , 4]);
                    startIndex = (page - 1) * limit;
                    return [4 /*yield*/, song_schema_1.default.aggregate([
                            {
                                $group: {
                                    _id: null,
                                    totalCount: { $sum: 1 },
                                },
                            },
                        ])];
                case 1:
                    totalQueryResult = _b.sent();
                    total = totalQueryResult.length > 0 ? totalQueryResult[0].totalCount : 0;
                    totalPages = Math.ceil(total / limit);
                    return [4 /*yield*/, song_schema_1.default.aggregate([
                            { $sample: { size: total } },
                            {
                                $project: {
                                    artist: 1,
                                    title: 1,
                                    link: 1,
                                    artwork: 1,
                                },
                            },
                        ])];
                case 2:
                    response = _b.sent();
                    data = response.slice(startIndex, startIndex + limit);
                    return [2 /*return*/, {
                            data: data,
                            total: total,
                            totalPages: totalPages,
                            currentPage: page,
                        }];
                case 3:
                    error_1 = _b.sent();
                    throw new server_1.TRPCError({
                        code: 'TIMEOUT',
                        message: 'Cant get songs at this moment, please try again later',
                    });
                case 4: return [2 /*return*/];
            }
        });
    });
};
exports.getAllSongs = getAllSongs;
var searchSongs = function (_a) {
    var limit = _a.limit, page = _a.page, searchText = _a.searchText;
    return __awaiter(void 0, void 0, void 0, function () {
        var startIndex, countQuery, response, _b, totalCount, data, total, totalPages, error_2;
        return __generator(this, function (_c) {
            switch (_c.label) {
                case 0:
                    _c.trys.push([0, 3, , 4]);
                    startIndex = (page - 1) * limit;
                    countQuery = song_schema_1.default.aggregate([
                        {
                            $match: {
                                $or: [
                                    { title: { $regex: searchText, $options: 'i' } },
                                    { artist: { $regex: searchText, $options: 'i' } }, // Case-insensitive artist search using regex
                                ],
                            },
                        },
                        {
                            $count: 'totalCount',
                        },
                    ]);
                    return [4 /*yield*/, song_schema_1.default.aggregate([
                            {
                                $match: {
                                    $or: [
                                        { title: { $regex: searchText, $options: 'i' } },
                                        { artist: { $regex: searchText, $options: 'i' } }, // Case-insensitive artist search using regex
                                    ],
                                },
                            },
                            {
                                $skip: startIndex,
                            },
                            {
                                $limit: limit,
                            },
                        ])];
                case 1:
                    response = _c.sent();
                    return [4 /*yield*/, Promise.all([countQuery, response])];
                case 2:
                    _b = _c.sent(), totalCount = _b[0], data = _b[1];
                    total = totalCount.length > 0 ? totalCount[0].totalCount : 0;
                    totalPages = Math.ceil(total / limit);
                    return [2 /*return*/, {
                            data: response,
                            total: total,
                            totalPages: totalPages,
                            currentPage: page,
                        }];
                case 3:
                    error_2 = _c.sent();
                    throw new server_1.TRPCError({
                        code: 'TIMEOUT',
                        message: 'Cant get songs at this moment, please try again later',
                    });
                case 4: return [2 /*return*/];
            }
        });
    });
};
exports.searchSongs = searchSongs;
var likeSong = function (_a) {
    var id = _a.id, userID = _a.userID;
    return __awaiter(void 0, void 0, void 0, function () {
        var liked, results, error_3;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    _b.trys.push([0, 6, , 7]);
                    liked = false;
                    return [4 /*yield*/, like_schema_1.default.findOne({
                            songId: id,
                            userId: userID,
                        })];
                case 1:
                    results = _b.sent();
                    if (!results) return [3 /*break*/, 3];
                    return [4 /*yield*/, like_schema_1.default.findOneAndDelete({
                            songId: id,
                            userId: userID,
                        })];
                case 2:
                    _b.sent();
                    return [3 /*break*/, 5];
                case 3:
                    liked = true;
                    return [4 /*yield*/, like_schema_1.default.create({
                            songId: id,
                            userId: userID,
                        })];
                case 4:
                    _b.sent();
                    _b.label = 5;
                case 5: return [2 /*return*/, {
                        liked: liked,
                    }];
                case 6:
                    error_3 = _b.sent();
                    throw new server_1.TRPCError({
                        code: 'TIMEOUT',
                        message: 'Cant get songs at this moment, please try again later',
                    });
                case 7: return [2 /*return*/];
            }
        });
    });
};
exports.likeSong = likeSong;
var getUserLikes = function (_a) {
    var userID = _a.userID;
    return __awaiter(void 0, void 0, void 0, function () {
        var results, error_4;
        return __generator(this, function (_b) {
            switch (_b.label) {
                case 0:
                    _b.trys.push([0, 2, , 3]);
                    return [4 /*yield*/, like_schema_1.default.find({
                            userId: userID,
                        })];
                case 1:
                    results = _b.sent();
                    return [3 /*break*/, 3];
                case 2:
                    error_4 = _b.sent();
                    throw new server_1.TRPCError({
                        code: 'TIMEOUT',
                        message: 'Cant get songs at this moment, please try again later',
                    });
                case 3: return [2 /*return*/];
            }
        });
    });
};
exports.getUserLikes = getUserLikes;
