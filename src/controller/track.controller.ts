import { Response, RequestHandler } from 'express'
import mongoose, { ObjectId } from 'mongoose'
import {tracksSchema} from '../schema/track.schema'

export const getRandom: RequestHandler = async (req, res) => {
    try {
        const { query } = req;
        const { search, page = 1, limit = 100 } = query;

        const tracks = await tracksSchema
            .aggregate([{ $sample: { size: parseInt(`${limit}`) } }])
            
        return res.status(200).json({tracks})
    } catch (error) {
        return res.status(303).json()
    }
}

export const search: RequestHandler = async (req, res) => {
    try {
        const { query } = req;
        const { search, page = 1, limit = 100 } = query;
        const skip = (parseInt(page as string) - 1) * parseInt(`${limit}`);
        
        if (search) {
            let query = {};

            query = {
              $or: [
                { artist: { $regex: search, $options: 'i' } },
                { title: { $regex: search, $options: 'i' } }
              ]
            };

            const totalCount = await tracksSchema.countDocuments();
            const totalPages = Math.ceil(totalCount / parseInt(`${limit}`));

            const tracks = await tracksSchema
                .find(query)
                .skip(skip)
                .limit(parseInt(`${limit}`))

            return res.status(200).json({
                tracks,
                totalItems: totalCount,
                totalPages: totalPages,
                currentPage: page,
            })
        }

        return res.status(303).json()
    } catch (error) {
        return res.status(303).json()
    }
}