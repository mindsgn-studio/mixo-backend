import { Schema, model } from 'mongoose'
import * as dotenv from 'dotenv'
dotenv.config()

const uploadSchema = new Schema(
    {
        link: {
            type: String,
            require: true,
            default: null,
        },
        processed: {
            type: Boolean,
            default: false,
        },
        uploadedBy: {
            type: String,
            require: true,
            default: null,
        },
    },
    {
        versionKey: false,
        timestamps: true,
    }
)

export default model('uploads', uploadSchema)
