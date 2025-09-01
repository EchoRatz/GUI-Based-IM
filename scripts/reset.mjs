// scripts/reset.mjs
// Run with: node scripts/reset.mjs
// Requires: npm install mongodb dotenv

import 'dotenv/config';
import { MongoClient } from 'mongodb';

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

    // Drop all docs from each collection
    for (const col of ["users", "conversations", "messages", "receipts"]) {
      const result = await db.collection(col).deleteMany({});
      console.log(`🧹 Cleared ${col} (${result.deletedCount} docs)`);
    }

    console.log("✅ chatdb fully reset (collections are empty now)");
  } catch (err) {
    console.error("❌ Error resetting data:", err);
  } finally {
    await client.close();
  }
}

run();
