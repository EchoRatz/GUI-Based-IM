// scripts/seed.mjs
// Run with: node scripts/seed.mjs
// Requires: npm install mongodb dotenv

import 'dotenv/config';           // <-- loads .env automatically
import { MongoClient } from "mongodb";

// Read MONGO_URI from .env
const uri = process.env.MONGO_URI;
if (!uri) {
  console.error("❌ Missing MONGO_URI in .env");
  process.exit(1);
}

const client = new MongoClient(uri);

async function run() {
  try {
    await client.connect();
    const db = client.db("chatdb");

    console.log("✅ Connected to Atlas");

    // --- Clear old data ---
    for (const col of ["users", "conversations", "messages", "receipts"]) {
      await db.collection(col).deleteMany({});
    }

    // --- Create Indexes ---
    await db.collection("users").createIndex({ username: 1 }, { unique: true });
    await db.collection("conversations").createIndex({ "members.user_id": 1 });
    await db
      .collection("messages")
      .createIndex({ conversation_id: 1, ts: -1 });
    await db
      .collection("receipts")
      .createIndex({ user_id: 1, conversation_id: 1 }, { unique: true });

    const now = Date.now();

    // --- Users ---
    await db.collection("users").insertMany([
      { _id: "u_1", username: "kan",  created_at: now, last_seen: now },
      { _id: "u_2", username: "mint", created_at: now, last_seen: now },
      { _id: "u_3", username: "pong", created_at: now, last_seen: now },
    ]);

    // --- Conversation ---
    await db.collection("conversations").insertOne({
      _id: "c_csclub",
      title: "CS Club",
      created_at: now,
      members: [
        { user_id: "u_1", role: "owner" },
        { user_id: "u_2", role: "member" },
        { user_id: "u_3", role: "member" },
      ],
    });

    // --- Messages ---
    const msgs = Array.from({ length: 10 }, (_, i) => ({
      _id: `m_${i + 1}`,
      conversation_id: "c_csclub",
      sender_id: i % 2 ? "u_2" : "u_1",
      ts: now - i * 60000,
      type: "text",
      body: `Seed message #${i + 1}`,
    }));
    await db.collection("messages").insertMany(msgs);

    // --- Receipts ---
    await db.collection("receipts").insertMany([
      {
        _id: "c_csclub#u_1",
        conversation_id: "c_csclub",
        user_id: "u_1",
        last_read_ts: now - 5 * 60000,
      },
      {
        _id: "c_csclub#u_2",
        conversation_id: "c_csclub",
        user_id: "u_2",
        last_read_ts: now - 7 * 60000,
      },
      {
        _id: "c_csclub#u_3",
        conversation_id: "c_csclub",
        user_id: "u_3",
        last_read_ts: now - 9 * 60000,
      },
    ]);

    console.log("✅ Seeded dummy data into chatdb");
  } catch (err) {
    console.error("❌ Error seeding data:", err);
  } finally {
    await client.close();
  }
}

run();
