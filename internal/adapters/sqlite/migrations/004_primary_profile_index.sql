CREATE UNIQUE INDEX one_primary_profile_per_business
    ON business_profiles (business_id)
    WHERE is_primary = 1;
