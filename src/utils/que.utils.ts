import { v4 as uuid } from 'uuid'
import { PassThrough } from 'stream'
import Throttle from 'throttle'
import { ffprobe } from '@dropb/ffprobe'
import ffprobeStatic from 'ffprobe-static'
import { readdir } from 'fs/promises'
import { createReadStream } from 'fs'
import { extname, join } from 'path'

ffprobe.path = ffprobeStatic.path

class Queue {
    tracks: any[]
    index: number
    clients: Map<any, any>
    bufferHeader: null
    currentTrack: any
    playing: any
    throttle: any
    stream: any
    constructor() {
        this.tracks = []
        this.index = 0
        this.clients = new Map()
        this.bufferHeader = null
    }

    current() {
        return this.tracks[this.index]
    }

    broadcast(chunk: any) {
        this.clients.forEach((client) => {
            client.write(chunk)
        })
    }

    addClient() {
        const id = uuid()
        const client = new PassThrough()

        this.clients.set(id, client)
        return { id, client }
    }

    removeClient(id: any) {
        this.clients.delete(id)
    }

    async loadTracks(dir: any) {
        let filenames = await readdir(dir)
        filenames = filenames.filter((filename) => extname(filename) === '.mp3')

        const filepaths = filenames.map((filename) => join(dir, filename))

        const promises = filepaths.map(async (filepath) => {
            const bitrate = await this.getTrackBitrate(filepath)
            return { filepath, bitrate }
        })

        this.tracks = await Promise.all(promises)
    }

    async getTrackBitrate(filepath: any) {
        const data = await ffprobe(filepath)
        const bitrate = data?.format?.bit_rate

        return bitrate ? parseInt(bitrate) : 128000
    }

    getNextTrack() {
        if (this.index >= this.tracks.length - 1) {
            this.index = 0
        }

        const track = this.tracks[this.index++]
        console.log(track)
        this.currentTrack = track

        return track
    }

    pause() {
        if (!this.started() || !this.playing) return
        this.playing = false
        this.throttle.removeAllListeners('end')
        this.throttle.end()
    }

    async resume() {
        await this.loadTracks('upload/audio')
        if (!this.started() || this.playing) return
        this.start()
    }

    started() {
        return this.stream && this.throttle && this.currentTrack
    }

    play(useNewTrack = false) {
        if (useNewTrack || !this.currentTrack) {
            this.getNextTrack()
            this.loadTrackStream()
            this.start()
        } else {
            this.resume()
        }
    }

    loadTrackStream() {
        const track = this.currentTrack
        if (!track) return
        this.stream = createReadStream(track.filepath)
    }

    async deleteAndPlay() {
        await this.loadTracks('upload/audio')
        this.play(true)
    }

    async start() {
        const track = this.currentTrack
        if (!track) return

        this.playing = true
        this.throttle = new Throttle(track.bitrate / 8)

        this.stream
            .pipe(this.throttle)
            .on('data', (chunk: any) => this.broadcast(chunk))
            .on('end', () => this.deleteAndPlay())
            .on('error', () => this.play(true))
    }
}

const queue = new Queue()
export default queue
