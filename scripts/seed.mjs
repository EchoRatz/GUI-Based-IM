// scripts/seed.mjs
// Run with: node scripts/seed.mjs
// Requires (project root): npm install mongodb dotenv

import 'dotenv/config';
import { MongoClient, ObjectId } from 'mongodb';

const uri = process.env.MONGO_URI;
if (!uri) {
  console.error("❌ Missing MONGO_URI in .env (include /chatdb)");
  process.exit(1);
}

const client = new MongoClient(uri);

async function run() {
  try {
    await client.connect();
    const db = client.db("chatdb");
    console.log("✅ Connected to Atlas (chatdb)");

    // 1) Clean (idempotent)
    for (const name of ["users", "conversations", "messages", "receipts"]) {
      await db.collection(name).deleteMany({});
    }

    // 2) Indexes
    await db.collection("users").createIndex({ username: 1 }, { unique: true });
    await db.collection("conversations").createIndex({ "members.user_id": 1 });
    await db.collection("messages").createIndex({ conversation_id: 1, ts: -1 });
    await db
      .collection("receipts")
      .createIndex({ user_id: 1, conversation_id: 1 }, { unique: true });

    const now = Date.now();

    // 3) USERS — _id are ObjectId (Mongo assigns)
    const insUsers = await db.collection("users").insertMany([
      { username: "kan",  created_at: now, last_seen: now },
      { username: "mint", created_at: now, last_seen: now },
      { username: "pong", created_at: now, last_seen: now },
    ]);
    const uKan  = insUsers.insertedIds["0"]; // ObjectId
    const uMint = insUsers.insertedIds["1"]; // ObjectId
    const uPong = insUsers.insertedIds["2"]; // ObjectId

    // 4) CONVERSATION — _id and members.user_id are ObjectId
    const convId = new ObjectId();
    await db.collection("conversations").insertOne({
      _id: convId,
      title: "CS Club",
      created_at: now,
      members: [
        { user_id: uKan,  role: "owner"  },
        { user_id: uMint, role: "member" },
        { user_id: uPong, role: "member" },
      ],
    });

    // 5) MESSAGES — conversation_id & sender_id are ObjectId
    const messages = Array.from({ length: 10 }, (_, i) => ({
      conversation_id: convId,
      sender_id: i % 2 ? uMint : uKan,
      ts: now - i * 60_000,
      type: "text",
      body: `Seed message #${i + 1}`,
    }));
    await db.collection("messages").insertMany(messages);

    // 6) RECEIPTS — user_id & conversation_id are ObjectId
    await db.collection("receipts").insertMany([
      { conversation_id: convId, user_id: uKan,  last_read_ts: now - 5 * 60_000 },
      { conversation_id: convId, user_id: uMint, last_read_ts: now - 7 * 60_000 },
      { conversation_id: convId, user_id: uPong, last_read_ts: now - 9 * 60_000 },
    ]);

    console.log("✅ Seeded with ObjectId everywhere");
    console.log({
      users: {
        kan:  uKan.toHexString(),
        mint: uMint.toHexString(),
        pong: uPong.toHexString(),
      },
      conversation_id: convId.toHexString(),
    });
  } catch (err) {
    console.error("❌ Error seeding data:", err);
  } finally {
    await client.close();
  }
}

run();
