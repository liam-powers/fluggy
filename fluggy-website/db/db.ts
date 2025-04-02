import { Pool } from "pg";

const pool = new Pool({
    connectionString: process.env.POSTGRES_URL,
    password: process.env.PGPASSWORD,
    ssl: {
        rejectUnauthorized: false,
    },
});

export async function getLeaderboardEntries(guildId: string) {
    try {
        const query = `
            SELECT * FROM users
            WHERE $1 = ANY(mutual_servers)
            ORDER BY elo ASC
        `;

        const result = await pool.query(query, [guildId]);
        return result.rows;
    } catch (error) {
        console.error("Error fetching leaderboard entries:", error);
        throw error;
    }
}

process.on("exit", () => {
    pool.end();
});
