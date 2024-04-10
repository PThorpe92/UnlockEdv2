<?php

declare(strict_types=1);

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;

class UserActivity extends Model
{
    use HasFactory;

    protected $fillable = [
        'user_id',
        'browser_name',
        'platform',
        'device',
        'ip',
        'clicked_url',
    ];

    protected $hidden = [
        'ip',
    ];

    public function user()
    {
        return $this->belongsTo(User::class);
    }
}
