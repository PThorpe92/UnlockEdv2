import {
    ProviderPlatform,
    ProviderPlatformState,
    ProviderPlatformType,
} from "@/common";
import { EyeIcon, EyeSlashIcon } from "@heroicons/react/24/solid";
import axios from "axios";
import { useState } from "react";
import { SubmitHandler, useForm } from "react-hook-form";

type ProviderInputs = {
    name: string;
    type: ProviderPlatformType;
    description: string;
    base_url: string;
    account_id: string;
    access_key: string;
    icon_url: string;
    state: ProviderPlatformState;
};

export default function EditProviderForm({
    onSuccess,
    provider,
}: {
    onSuccess: Function;
    provider: ProviderPlatform;
}) {
    const [errorMessage, setErrorMessage] = useState("");
    const [showAdditionalFields, setShowAdditionalFields] = useState(false);
    const [showAccessKey, setShowAccessKey] = useState(false);
    const [accessKey, setAccessKey] = useState("");
    const {
        register,
        handleSubmit,
        reset,
        formState: { errors },
    } = useForm<ProviderInputs>({
        defaultValues: {
            name: provider.name,
            description: provider.description,
            type: provider.type,
            base_url: provider.base_url,
            account_id: provider.account_id,
            icon_url: provider.icon_url,
            state: provider.state,
        },
    });

    const getAccessKey = async () => {
        if (showAccessKey) {
            setShowAccessKey(false);
            return;
        }
        if (accessKey) {
            setShowAccessKey(true);
            return;
        }
        try {
            const response = await axios.get(
                `/api/v1/provider-platforms/${provider?.id}?show_key=true`,
            );
            setAccessKey(response.data.data["access_key"]);
            setShowAccessKey(true);
        } catch (error: any) {
            setErrorMessage(error.response.data.message);
        }
    };

    function diffFormData(formData: any, currentUserData: any) {
        const changes: Partial<ProviderPlatform> = {};
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

    const onSubmit: SubmitHandler<ProviderInputs> = async (data) => {
        const cleanData = diffFormData(data, provider);
        try {
            setErrorMessage("");
            await axios.patch(
                `/api/v1/provider-platforms/${provider?.id}`,
                cleanData,
            );
            onSuccess();
            reset();
        } catch (error: any) {
            setErrorMessage(error.response.data.message);
        }
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <label className="form-control">
                <div className="label">
                    <span className="label-text">Provider Name:</span>
                </div>
                <input
                    type="text"
                    className="input input-bordered w-full"
                    {...register("name", {
                        required: "Provider name is required",
                        maxLength: {
                            value: 25,
                            message:
                                "Provider name should be 25 characters or less",
                        },
                    })}
                />
                <div className="text-error text-sm">
                    {errors.name && errors.name?.message}
                </div>
            </label>

            <label className="form-control">
                <div className="label">
                    <span className="label-text">Description</span>
                </div>
                <textarea
                    className="textarea textarea-bordered w-full"
                    {...register("description", {
                        required: "Description is required",
                        maxLength: {
                            value: 255,
                            message:
                                "Description should be 255 characters or less",
                        },
                    })}
                />
                <div className="text-error text-sm">
                    {errors.description && errors.description?.message}
                </div>
            </label>

            <label className="form-control w-full">
                <div className="label">
                    <span className="label-text">Type</span>
                </div>
                <select
                    className="select select-bordered"
                    {...register("type", { required: "Type is required" })}
                >
                    {Object.keys(ProviderPlatformType).map((key) => (
                        <option
                            key={key}
                            value={
                                ProviderPlatformType[
                                    key as keyof typeof ProviderPlatformType
                                ]
                            }
                        >
                            {
                                ProviderPlatformType[
                                    key as keyof typeof ProviderPlatformType
                                ]
                            }
                        </option>
                    ))}
                </select>
                <div className="text-error text-sm">
                    {errors.type && errors.type?.message}
                </div>
            </label>

            <label className="form-control w-full">
                <div className="label">
                    <span className="label-text">State </span>
                </div>
                <select
                    className="select select-bordered"
                    {...register("state", { required: "State is required" })}
                >
                    {Object.keys(ProviderPlatformState).map((key) => (
                        <option
                            key={key}
                            value={
                                ProviderPlatformState[
                                    key as keyof typeof ProviderPlatformState
                                ]
                            }
                        >
                            {
                                ProviderPlatformState[
                                    key as keyof typeof ProviderPlatformState
                                ]
                            }
                        </option>
                    ))}
                </select>
                <div className="text-error text-sm">
                    {errors.state && errors.state?.message}
                </div>
            </label>

            {/* Button to toggle additional fields */}
            <div className="pt-4">
                <button
                    className="btn btn-ghost"
                    type="button"
                    onClick={() =>
                        setShowAdditionalFields(!showAdditionalFields)
                    }
                >
                    {showAdditionalFields ? "Show Less" : "Show More"}
                </button>
            </div>

            <div className={showAdditionalFields ? "contents" : "hidden"}>
                <label className="form-control">
                    <div className="label">
                        <span className="label-text">Base URL</span>
                    </div>
                    <input
                        type="text"
                        className="input input-bordered w-full"
                        {...register("base_url", {
                            required: "Base URL is required",
                        })}
                    />
                    <div className="text-error text-sm">
                        {errors.base_url && errors.base_url?.message}
                    </div>
                </label>

                <label className="form-control">
                    <div className="label">
                        <span className="label-text">Account Id</span>
                    </div>
                    <input
                        type="text"
                        className="input input-bordered w-full"
                        {...register("account_id", {
                            required: "Account Id is required",
                        })}
                    />
                    <div className="text-error text-sm">
                        {errors.account_id && errors.account_id?.message}
                    </div>
                </label>
                <label className="form-control">
                    <div className="label">
                        <span className="label-text">Access Key</span>
                    </div>
                    <div className="relative">
                        {showAccessKey ? (
                            <input
                                type="text"
                                className="input input-bordered w-full pr-10"
                                value={accessKey}
                                {...register("access_key", {
                                    required: "Access Key is required",
                                    value: accessKey,
                                    onChange: (e) =>
                                        setAccessKey(e.target.value),
                                })}
                            />
                        ) : (
                            <input
                                type="password"
                                className="input input-bordered w-full"
                                value="**********"
                                readOnly // Make the input read-only when showAccessKey is false
                            />
                        )}
                        {showAccessKey ? (
                            <EyeSlashIcon
                                className="w-4 z-10 top-4 right-4 absolute"
                                onClick={() => {
                                    console.log(accessKey),
                                        setAccessKey(accessKey),
                                        setShowAccessKey(false);
                                }}
                            />
                        ) : (
                            <EyeIcon
                                className="w-4 z-10 top-4 right-4 absolute"
                                onClick={getAccessKey}
                            />
                        )}
                    </div>
                    <div className="text-error text-sm">
                        {errors.access_key && errors.access_key?.message}
                    </div>
                </label>
                <label className="form-control">
                    <div className="label">
                        <span className="label-text">Icon URL</span>
                    </div>
                    <input
                        type="text"
                        className="textarea textarea-bordered w-full"
                        {...register("icon_url")}
                    />
                    <div className="text-error text-sm">
                        {errors.icon_url && errors.icon_url?.message}
                    </div>
                </label>
            </div>

            <label className="form-control pt-4">
                <input className="btn btn-primary" type="submit" value="Save" />
                <div className="text-error text-center pt-2">
                    {errorMessage}
                </div>
            </label>
        </form>
    );
}
