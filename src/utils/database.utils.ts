import mongoose from "mongoose";
import pino from "pino";
import * as dotenv from "dotenv";
dotenv.config();
const logger = pino({ level: "info" });

(async () => {
  try {
    const db = await mongoose.connect(`${process.env.MONGO}`, {
      dbName: 'mixo',
    });
    console.log("Database is connected to: ", db.connection.name);
  } catch (error) {
    console.log("Databse Error");
    console.error(error);
  }
})();