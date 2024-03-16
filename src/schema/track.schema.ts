import { Schema, model } from "mongoose";

const trackSchema = new Schema(
  {
    title: { 
      type: String, 
      required: true 
    },
    artist: {
      type: String, 
      required: true 
    },
    link:{
      type: String, 
      required: true 
    },
    trackNumber: {
      type: String, 
      required: false 
    },
    albumTitle:{
      type: String, 
      required: false 
    },
    coverArt: {
      type: String, 
      required: true
    },
  },
  {
    versionKey: false,
    timestamps: true,
  }
);

const tracksSchema = model('tracks', trackSchema);

export { tracksSchema };