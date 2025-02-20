// import { useState } from 'react';
import useSWR from 'swr';
import { AxiosError } from 'axios';
import { ResidentEngagementProfile, ServerResponseOne } from '@/common';
// import { ResponsiveContainer } from 'recharts';
// import StatsCard from './StatsCard';
import NewEngagementRateGraph from '@/Components/EngagementRateGraph';
import { ResponsiveContainer } from 'recharts';
// import { useAuth } from '@/useAuth';
import StatsCard from '@/Components/StatsCard';
import { UserCircleIcon } from '@heroicons/react/24/outline';
import { useParams } from 'react-router-dom';
// TODO: figure out how to pass the studentId to this page
const StudentProfile = () => {
    // const { user } = useAuth();
    // const [resetCache, setResetCache] = useState(false);
    const { user_id } = useParams<{ user_id: string }>();
    const uid = Number(user_id);
    const { data, error, isLoading } = useSWR<
        ServerResponseOne<ResidentEngagementProfile>,
        AxiosError
    >(`/api/users/${uid}/profile`);
    const metrics = data?.data;

    // const { data: facilitiesData } =
    //     useSWR<ServerResponseOne<Facility[]>>('/api/facilities');

    // useEffect(() => {
    //     void mutate();
    // }, [facility, days, resetCache]);

    // const facilities = facilitiesData?.data;

    // const formattedDate =
    //     metrics && new Date(metrics.last_cache).toLocaleString('en-US', {});

    // const totalUsers =
    //     (metrics?.data.total_residents ?? 0) +
    //     (metrics?.data.total_admins ?? 0);
    return (
        <div className="overflow-x-hidden">
            {error && <div>Error loading data</div>}
            {!data || (isLoading && <div>Loading...</div>)}
            {data && metrics && (
                <>
                    {/* <div className="flex items-end justify-between pb-4">
                         <div className="flex flex-row gap-4">
                             
                         </div>
                         <div>
                             <p className="label label-text text-grey-3">
                                 Last updated:
                             </p>
                             <button
                                 className="button justify-self-end"
                                 // onClick={() => setResetCache(!resetCache)}
                             >
                                 Refresh Data
                             </button>
                         </div>
                     </div> */}

                    <div className="flex flex-row gap-6">
                        <div className="w-1/5 flex flex-col gap-4">
                            <div className="card card-row-padding overflow-hidden text-2xl items-center">
                                <UserCircleIcon className="w-1/4 h-1/4" />
                                Michael Jackson
                                {/* {user?.name_first +" "+user?.name_last} */}
                            </div>
                        </div>
                        <div className="w-4/5 flex flex-col gap-4">
                            <div className="card card-row-padding overflow-hidden">
                                {/* <h1 className="">{user?.name_first}'s Recent Activity</h1> */}
                                <h1 className="">Mike's Recent Activity</h1>
                                <div className=" items-stretch gap-12 px-10 pt-10 ">
                                    <div className="w-full h-[240px] overflow-visible">
                                        <ResponsiveContainer
                                            className="w-full h-full overflow-visible"
                                            width="100%"
                                            height="100%"
                                            debounce={500}
                                        >
                                            <NewEngagementRateGraph
                                                peak_login_times={
                                                    metrics?.login_engagement
                                                        .peak_login_times ?? []
                                                }
                                                viewType={'daily'}
                                            />
                                        </ResponsiveContainer>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="grid grid-cols-3 gap-4 mb-6 mt-6">
                        <StatsCard
                            title="Days Active"
                            number={metrics.activity_engagement.total_hours_active_monthly.toString()}
                            label="Days Active This Month"
                            tooltip="Total number of days resident has been active in UnlockedEd"
                        />
                        <StatsCard
                            title="Average Hours"
                            number={metrics.activity_engagement.total_hours_active_weekly.toString()}
                            label={'AVG Hours PER Week'}
                            tooltip={
                                'Average number of hours resident is logged in to UnlockedEd'
                            }
                        />
                        <StatsCard
                            title="Total Hours"
                            number={metrics.activity_engagement.total_hours_engaged.toString()}
                            label={'Total Hours This Week'}
                            tooltip={
                                'Total number of hours resident was logged in to UnlockedEd this week'
                            }
                        />
                    </div>
                </>
            )}
        </div>
    );
};

export default StudentProfile;
