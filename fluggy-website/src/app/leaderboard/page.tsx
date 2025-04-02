import { getLeaderboardEntries } from "../../../db/db";

interface PageProps {
    searchParams: {
        guildId?: string;
    };
}

export default async function Leaderboard({ searchParams }: PageProps) {
    const guildId = (await searchParams).guildId;
    if (!guildId) {
        return <div>Error: No guildId provided in URL query params.</div>;
    }

    // 1. using guildId, grab leaderboard data for this guild
    const leaderboardData = await getLeaderboardEntries(guildId);

    console.log("leaderboardData:", leaderboardData);
    return (
        <div>
            <p>Value of guildId: {guildId}</p>
            {/* <p>Leaderboard Data: {leaderboardData}</p> */}
        </div>
    );
}
