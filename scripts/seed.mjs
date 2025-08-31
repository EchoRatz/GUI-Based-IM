// scripts/seed.mjs
// Run with: node scripts/seed.mjs
// Requires: npm install mongodb dotenv

import 'dotenv/config';
import { MongoClient, ObjectId } from 'mongodb';

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

    // --- Clean (idempotent) ---
    for (const col of ["users", "conversations", "messages", "receipts"]) {
      await db.collection(col).deleteMany({});
    }

    // --- Indexes (safe if already exist) ---
    await db.collection("users").createIndex({ username: 1 }, { unique: true });
    await db.collection("conversations").createIndex({ "members.user_id": 1 });
    await db.collection("messages").createIndex({ conversation_id: 1, ts: -1 });
    await db.collection("receipts").createIndex(
      { user_id: 1, conversation_id: 1 },
      { unique: true }
    );

    const now = Date.now();

    // --- USERS ---
    // Let Mongo create ObjectIDs for _id
    const usersInsert = await db.collection("users").insertMany([
      { username: "kan",  created_at: now, last_seen: now },
      { username: "mint", created_at: now, last_seen: now },
      { username: "pong", created_at: now, last_seen: now },
    ]);

    // Map usernames -> ObjectID hex strings (for references)
    const uid = {
      kan:  usersInsert.insertedIds["0"].toHexString(),
      mint: usersInsert.insertedIds["1"].toHexString(),
      pong: usersInsert.insertedIds["2"].toHexString(),
    };

    // --- CONVERSATION ---
    const convId = new ObjectId();
    const convHex = convId.toHexString();

    await db.collection("conversations").insertOne({
      _id: convId,                 // store as real ObjectId
      title: "CS Club",
      created_at: now,
      members: [
        { user_id: uid.kan,  role: "owner"  },
        { user_id: uid.mint, role: "member" },
        { user_id: uid.pong, role: "member" }
      ],
    });

    // --- MESSAGES ---
    // conversation_id and sender_id stored as hex string references (consistent with your Go JSON)
    const messages = Array.from({ length: 10 }, (_, i) => ({
      conversation_id: convHex,
      sender_id: i % 2 ? uid.mint : uid.kan,
      ts: now - i * 60_000,
      type: "text",
      body: `Seed message #${i + 1}`,
    }));
    await db.collection("messages").insertMany(messages);

    // --- RECEIPTS ---
    await db.collection("receipts").insertMany([
      { conversation_id: convHex, user_id: uid.kan,  last_read_ts: now - 5 * 60_000 },
      { conversation_id: convHex, user_id: uid.mint, last_read_ts: now - 7 * 60_000 },
      { conversation_id: convHex, user_id: uid.pong, last_read_ts: now - 9 * 60_000 },
    ]);

    console.log("✅ Seeded dummy data into chatdb with ObjectIDs (refs as hex strings)");
    console.log({ users: uid, conversation_id: convHex });
  } catch (err) {
    console.error("❌ Error seeding data:", err);
  } finally {
    await client.close();
  }
}

run();
