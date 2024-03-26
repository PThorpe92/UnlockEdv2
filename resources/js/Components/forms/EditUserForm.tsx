import { User, UserRole } from "@/common";
import axios from "axios";
import { useState } from "react";
import { useForm, SubmitHandler } from "react-hook-form";
import { CloseX } from "../inputs/CloseX";

type Inputs = {
    name_first: string;
    name_last: string;
    username: string;
    role: UserRole;
    email: string;
};

export default function EditUserForm({
    onSuccess,
    user,
}: {
    onSuccess: () => void;
    user: null | User;
}) {
    if (user === null) {
        return <div>No user defined!</div>;
    }

    const [errorMessage, setErrorMessage] = useState("");

    const {
        register,
        handleSubmit,
        formState: { errors },
    } = useForm<Inputs>();

    function diffFormData(formData: any, currentUserData: any) {
        const changes: Partial<User> = {};
        Object.keys(formData).forEach((key) => {
            if (
                formData[key] !== currentUserData[key] &&
                formData[key] !== undefined
            ) {
                changes[key] = formData[key];
            }
        });
        return changes;
    }

    const onSubmit: SubmitHandler<Inputs> = async (data) => {
        try {
            setErrorMessage("");
            const cleanData = diffFormData(data, user);
            await axios.patch(`/api/v1/users/${user.id}`, cleanData);

            onSuccess();
        } catch (error: any) {
            setErrorMessage(error.response.data.message);
        }
    };

    return (
        <>
            <form method="dialog">
                <CloseX close={() => onSuccess()} />
            </form>
            <form onSubmit={handleSubmit(onSubmit)}>
                <label className="form-control">
                    <div className="label">
                        <span className="label-text">First Name</span>
                    </div>
                    <input
                        type="text"
                        className="input input-bordered w-full"
                        defaultValue={user.name_first}
                        {...register("name_first", {
                            required: "First name is required",
                            maxLength: {
                                value: 25,
                                message:
                                    "First name should be 25 characters or less",
                            },
                        })}
                    />
                    <div className="text-error text-sm">
                        {errors.name_first && errors.name_first?.message}
                    </div>
                </label>
                <label className="form-control">
                    <div className="label">
                        <span className="label-text">Last Name</span>
                    </div>
                    <input
                        type="text"
                        className="input input-bordered w-full"
                        defaultValue={user.name_last}
                        {...register("name_last", {
                            required: "Last name is required",
                            maxLength: {
                                value: 25,
                                message:
                                    "Last name should be 25 characters or less",
                            },
                        })}
                    />
                    <div className="text-error text-sm">
                        {errors.name_last && errors.name_last?.message}
                    </div>
                </label>

                <label className="form-control">
                    <div className="label">
                        <span className="label-text">Username</span>
                    </div>
                    <input
                        type="text"
                        className="input input-bordered w-full"
                        defaultValue={user.username}
                        {...register("username", {
                            required: "Username is required",
                            maxLength: {
                                value: 50,
                                message:
                                    "Username should be 50 characters or less",
                            },
                        })}
                    />
                    <div className="text-error text-sm">
                        {errors.username && errors.username?.message}
                    </div>
                </label>

                <label className="form-control">
                    <div className="label">
                        <span className="label-text">Email</span>
                    </div>
                    <input
                        type="text"
                        className="input input-bordered w-full"
                        defaultValue={user.email}
                        {...register("email", {
                            maxLength: {
                                value: 50,
                                message:
                                    "Email should be 50 characters or less",
                            },
                        })}
                    />
                    <div className="text-error text-sm">
                        {errors.email && errors.email?.message}
                    </div>
                </label>

                <label className="form-control w-full">
                    <div className="label">
                        <span className="label-text">Role</span>
                    </div>
                    <select
                        className="select select-bordered"
                        defaultValue={user.role}
                        {...register("role", { required: "Role is required" })}
                    >
                        <option value="student">Student</option>
                        <option value="admin">Admin</option>
                    </select>
                    <div className="text-error text-sm">
                        {errors.role && errors.role?.message}
                    </div>
                </label>

                <label className="form-control pt-4">
                    <input className="btn btn-primary" type="submit" />
                    <div className="text-error text-center pt-2">
                        {errorMessage}
                    </div>
                </label>
            </form>
        </>
    );
}
