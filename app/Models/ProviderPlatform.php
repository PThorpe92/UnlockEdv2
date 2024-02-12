<?php

namespace App\Models;

use App\Enums\ProviderPlatformState;
use App\Enums\ProviderPlatformType;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;

class ProviderPlatform extends Model
{
    use HasFactory;

    protected $fillable = [
        'type',
        'name',
        'description',
        'icon_url',
        'account_id',
        'access_key',
        'base_url',
        'state',
        'external_auth_provider_id',
    ];

    /**
     * The attributes that should be cast.
     *
     * @var array<string, string>
     */
    protected $casts = [
        'type' => ProviderPlatformType::class,
        'state' => ProviderPlatformState::class,
    ];

    // public function hashAccessKey()
    // {
    //     $accessKey = $this->access_key;
    //     $hashedAccessKey = Hash::make($accessKey);
    //     $this->access_key = $hashedAccessKey;
    //     $this->save();
    // }

    public function providerUserMappings()
    {
        return $this->hasMany('App\Models\ProviderUserMapping');
    }
}
