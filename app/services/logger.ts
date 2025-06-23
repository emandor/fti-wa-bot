import pino from "pino";
import { multistream } from "pino-multi-stream";
import fs from "fs";

// Create a write stream for logs file
const logFileStream = fs.createWriteStream("./logs/full.log", { flags: "a" });

export const logger = pino(
  {
    level: "debug",
    formatters: {
      level(label) {
        return { level: label };
      },
    },
    timestamp: pino.stdTimeFunctions.isoTime,
  },
  multistream([
    { stream: process.stdout }, // pretty console output
    { stream: logFileStream }, // full log file
  ]),
);
