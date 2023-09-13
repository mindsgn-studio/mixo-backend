import { Router } from 'express'
import multer from 'multer'
import uploadSchema from '../schema/upload.schema'
import { v4 as uuid } from 'uuid'

const uploadRouter = Router()
const storage = multer.diskStorage({
    destination: (req, file, cb) => {
        cb(null, 'upload/audio/')
    },
    filename: async (req, file, cb) => {
        try {
            const { user } = req.query

            const newFile: string = `${uuid()}.mp3`
            cb(null, `${newFile}`)

            const saveUpload = new uploadSchema({
                link: `${process.env.LINK}/${newFile}`,
                uploadedBy: user,
            })

            await saveUpload.save()
        } catch (error: any) {
            console.error(error.message)
        }
    },
})

const fileFilter = (req: any, file: any, cb: any) => {
    if (file.mimetype === 'audio/mpeg') {
        cb(null, true)
    } else {
        cb(new Error('Invalid file type. Only MP3 files are allowed.'))
    }
}

const upload = multer({ storage: storage, fileFilter: fileFilter })

uploadRouter.post('/upload/audio', upload.any(), async (req, res) => {
    try {
        const { page = 1, limit = 10 } = req.query
        const skipCount = (parseInt(`${page}`) - 1) * parseInt(`${limit}`)

        const response = await uploadSchema
            .find({}, { projection: { _id: 0, createdAt: 0, updatedAt: 0 } })
            .skip(skipCount)
            .limit(parseInt(`${limit}`))

        return res.status(200).json({
            data: response,
            error: true,
        })
    } catch {
        return res.status(303).json({
            data: [],
            error: true,
        })
    }
})

export { uploadRouter }
