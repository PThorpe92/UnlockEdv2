<?php

declare(strict_types=1);

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class UpdateCourseRequest extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return $this->user()->isAdmin();
    }

    /**
     * Get the validation rules that apply to the request.
     *
     * @return array<string, \Illuminate\Contracts\Validation\ValidationRule|array<mixed>|string>
     */
    public function rules(): array
    {
        return [
            'external_resource_id' => 'nullable|string|max:255',
            'external_course_name' => 'nullable|string|max:255',
            'external_course_code' => 'nullable|string|max:255',
            'description' => 'nullable|string|max:255',
            'img_url' => 'nullable|url|max:255',
        ];
    }
}
