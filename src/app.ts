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
import helmet from 'helmet'
import * as path from 'path'
import { trackRouter } from './routes/track.route.'
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
app.use(trackRouter)
// app.use()

app.use(
    '/api',
    createExpressMiddleware({
        router: apiRoute,
        createContext,
    })
)

app.use('/', (_, res) => {
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
        const { body, query } = req
        const { address } = query || req

        if (address)
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
        connectDatabase()
    })
})()

export type ApiRoute = typeof apiRoute
