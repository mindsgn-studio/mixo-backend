import express from 'express'
import { createExpressMiddleware } from '@trpc/server/adapters/express'
import dotenv from 'dotenv'
import cors from 'cors'
import { createContext } from './utils/trpc.utils'
import { database as connectDatabase } from './utils/database.utils'
import { apiRoute } from './routes/api.route'
import { renderTrpcPanel } from 'trpc-panel'
import queue from './utils/que.utils'
import { startup } from './utils/startup'
import multer from 'multer'
import { v4 as uuid } from 'uuid'
import helmet from 'helmet'
import uploadSchema from './schema/upload.schema'
import { Stream } from 'stream'
const path = require('path')
const fs = require('fs')
const MP3Tag = require('mp3tag.js')
const audioDirectory = path.join(__dirname, '..', 'upload', 'audio')

let streaming: boolean = false

startup()
dotenv.config()

const app = express()

app.use(
    helmet({
        crossOriginResourcePolicy: false,
    })
)
app.use(express.json())
app.use(express.urlencoded({ extended: false }))
app.use(cors())
app.use('/audio', express.static(audioDirectory))

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

app.post('/upload/audio', upload.any(), async (req, res) => {
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

app.use(
    '/api',
    createExpressMiddleware({
        router: apiRoute,
        createContext,
    })
)

app.use('/panel', (_, res) => {
    return res.send(
        renderTrpcPanel(apiRoute, { url: 'http://localhost:8080/api' })
    )
})
;(async () => {
    const getFiles = async () => {
        await queue.loadTracks('upload/audio')
    }

    const play = async () => {
        queue.play()
    }

    app.get('/stream', (req, res) => {
        if (streaming) {
            const { id, client } = queue.addClient()

            res.set({
                'Content-Type': 'audio/mp3',
                'Transfer-Encoding': 'chunked',
            }).status(200)
            client.pipe(res)
            req.on('close', () => {
                queue.removeClient(id)
            })
        } else {
            res.send(200).json({})
        }
    })

    app.listen(process.env.PORT, () => {
        console.log(`🚀 Server listening on port ${process.env.PORT}`)
        connectDatabase()
    })
})()

export type ApiRoute = typeof apiRoute
