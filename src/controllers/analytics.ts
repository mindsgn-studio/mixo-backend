import { TRPCError } from '@trpc/server'
import dotenv from 'dotenv'
dotenv.config()

export const postAnaltics = async () => {
    try {
    } catch (error: any) {
        throw new TRPCError({
            code: 'TIMEOUT',
            message: 'Cant get songs at this moment, please try again later',
        })
    }
}
