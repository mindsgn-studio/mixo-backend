import { Router } from 'express'
import * as song from '../controllers/track.controler'

const trackRouter = Router()

trackRouter.get('/song/status', song.getStatus)
trackRouter.get('/song/search/:title', song.getSearch)
trackRouter.get('/song/random', song.getRandom)
trackRouter.get('/song/all', song.getAllTracks)
trackRouter.post('/song/like/:id/:userID', song.postLike)
trackRouter.get('/song/like/:id/:userID', song.getLike)
trackRouter.post('/song/stream/:id/:userID', song.postStream)

// always on the end
trackRouter.get('/song/:id', song.getTrackbyID)

export { trackRouter }
