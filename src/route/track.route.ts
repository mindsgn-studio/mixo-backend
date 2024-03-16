import { Router } from 'express'
import * as track from '../controller/track.controler'

const router = Router();

router.get('/track/search', track.search);
router.get('/track/random', track.getRandom);

export default router;