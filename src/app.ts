import dotenv from "dotenv";
import express from "express";
import helmet from "helmet";
import { createServer } from "http"; 
import rateLimit from "express-rate-limit";
import trackRoute from "./route/track.route";
import "./utils/database.utils";

dotenv.config();

const limiter = rateLimit({
  windowMs: 15 * 60 * 1000,
  max: 100,
});

const app = express();
const server = createServer(app);

app.use(limiter);
app.use(helmet());
app.use(express.json());
app.use(express.urlencoded({ extended: false }));
app.use(trackRoute);

server.listen(process.env.PORT, () => {
  console.log(`Server listening on port ${process.env.PORT}`);
});

export default app;