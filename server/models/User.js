const mongoose = require("mongoose");

const userSchema = new mongoose.Schema({
  username: {
    type: String,
    required: true,
    unique: true,
    trim: true,
  },
  password: {
    type: String,
    required: true,
  },
  connectedAt: {
    type: Date,
    default: Date.now,
  },
  isOnline: {
    type: Boolean,
    default: true,
  },
});

module.exports = mongoose.model("User", userSchema);
