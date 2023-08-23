import express from 'express'
import { createExpressMiddleware } from '@trpc/server/adapters/express'
import dotenv from 'dotenv'
import cors from 'cors'
import { createContext } from './utils/trpc.utils'
import { database as connectDatabase } from './utils/database.utils'
import { apiRoute } from './routes/api.route'
import { renderTrpcPanel } from 'trpc-panel'
import queue from './utils/que.utils'

dotenv.config()
const app = express()
app.use(cors())

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
        const { id, client } = queue.addClient()

        res.set({
            'Content-Type': 'audio/mp3',
            'Transfer-Encoding': 'chunked',
        }).status(200)
        client.pipe(res)
        req.on('close', () => {
            queue.removeClient(id)
        })
    })

    app.listen(process.env.PORT, () => {
        console.log(`🚀 Server listening on port ${process.env.PORT}`)
        connectDatabase()
    })
})()

export type ApiRoute = typeof apiRoute
