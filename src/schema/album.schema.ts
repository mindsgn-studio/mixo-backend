import { Schema, model } from "mongoose";

const albumSchema = new Schema(
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

const albumsSchema = model('albums', albumSchema);

export { albumsSchema };