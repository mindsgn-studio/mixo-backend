"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var mongoose_1 = require("mongoose");
var dotenv = require("dotenv");
dotenv.config();
var songSchema = new mongoose_1.Schema({
    artist: {
        type: [String],
        require: true,
    },
    title: {
        type: String,
        require: true,
    },
    link: {
        type: String,
        require: true,
    },
    artwork: {
        type: String,
        require: false,
        default: null,
    },
}, {
    versionKey: false,
    timestamps: true,
});
exports.default = (0, mongoose_1.model)('songs', songSchema);
