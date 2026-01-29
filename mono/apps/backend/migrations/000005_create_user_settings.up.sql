-- ============================================================================
-- USER SETTINGS
-- ============================================================================
-- Stores user preferences and feature toggles for personalization and control

CREATE TABLE llm_api.user_settings (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    
    -- Memory Configuration stored as JSON for flexibility
    memory_config JSONB NOT NULL DEFAULT '{
        "enabled": true,
        "observe_enabled": true,
        "inject_user_core": true,
        "inject_semantic": true,
        "inject_episodic": false,
        "max_user_items": 3,
        "max_project_items": 5,
        "max_episodic_items": 3,
        "min_similarity": 0.75
    }',
    
    -- Profile Settings
    profile_settings JSONB NOT NULL DEFAULT '{
        "base_style": "Friendly",
        "custom_instructions": "",
        "nick_name": "",
        "occupation": "",
        "more_about_you": ""
    }',
    
    -- Advanced Settings
    advanced_settings JSONB NOT NULL DEFAULT '{
        "web_search": false,
        "code_enabled": false
    }',
    
    -- Other Feature Toggles
    enable_trace BOOLEAN NOT NULL DEFAULT false,
    enable_tools BOOLEAN NOT NULL DEFAULT true,
    
    -- Preferences stored as flexible JSON for future extensions
    preferences JSONB NOT NULL DEFAULT '{}',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_user_settings_user FOREIGN KEY (user_id) REFERENCES llm_api.users(id) ON DELETE CASCADE,
    CONSTRAINT ux_user_settings_user_id UNIQUE (user_id)
);

CREATE INDEX idx_user_settings_user_id ON llm_api.user_settings(user_id);

-- Add helpful comments
COMMENT ON TABLE llm_api.user_settings IS 'User preferences and feature toggles with JSONB columns for flexible configuration';
COMMENT ON COLUMN llm_api.user_settings.memory_config IS 'Memory configuration: enabled, auto_inject, observe_enabled, inject flags, retrieval limits, similarity threshold';
COMMENT ON COLUMN llm_api.user_settings.profile_settings IS 'User profile information: base_style, custom_instructions, nick_name (alias nickname), occupation, more_about_you';
COMMENT ON COLUMN llm_api.user_settings.advanced_settings IS 'Advanced features: web_search, code_enabled';
