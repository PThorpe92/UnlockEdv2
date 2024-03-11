<?php

namespace App\Http\Controllers\v1\Actions;

use App\Http\Controllers\Controller;
use App\Http\Requests\AdminRequest;
use App\Http\Resources\CourseResource;
use App\Models\Course;
use App\Services\CanvasServices;

class StoreCanvasCoursesController extends Controller
{
    public function __invoke(AdminRequest $request)
    {
        try {
            $canvasService = CanvasServices::byProviderId($request->provider_platform_id);
        } catch (\Exception) {
            return response()->json(['message' => 'Provider not found'], 404);
        }
        // list all the courses in the account
        $canvasCourses = $canvasService->listCourses();
        $courseCollection = collect();
        foreach ($canvasCourses as $course) {
            $request->merge([
                'external_resource_id' => $course->id, 'external_course_name' => $course->name, 'provider_platform_id' => $request->provider_platform_id,
                'description' => $course->public_description, 'img_url' => $course->course_image, 'external_course_code' => $course->course_code,
            ]);
            $validated = $request->validate([
                'provider_platform_id' => 'required|exists:provider_platforms,id',
                'external_resource_id' => 'required|string|max:255|unique:courses,external_resource_id',
                'external_course_name' => 'required|string|max:255',
                'description' => 'required|string|max:255',
                'img_url' => 'nullable|string|max:255',
                'external_course_code' => 'required|string|max:255',
            ]);
            $courseCollection->push(Course::create($validated));
        }

        return CourseResource::collection($courseCollection);
    }
}
