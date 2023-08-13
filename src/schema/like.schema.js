"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var mongoose_1 = require("mongoose");
var dotenv = require("dotenv");
dotenv.config();
var likeSchema = new mongoose_1.Schema({
    songId: {
        type: String,
        require: false,
        default: null,
    },
    userId: {
        type: String,
        require: false,
        default: null,
    },
    date: {
        type: Date,
        require: false,
        default: Date.now,
    },
}, {
    versionKey: false,
    timestamps: true,
});
exports.default = (0, mongoose_1.model)('likes', likeSchema);
