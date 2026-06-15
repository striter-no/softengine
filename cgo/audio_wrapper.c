// audio_wrapper.c
#define MINIAUDIO_IMPLEMENTATION
#include "./vendor/miniaudio.h"
#include "audio_wrapper.h"

ma_engine engine;

int audio_init() {
    ma_result result = ma_engine_init(NULL, &engine);
    if (result != MA_SUCCESS) return -1;

    ma_engine_listener_set_position(&engine, 0, 0.0f, 0.0f, 0.0f);
    ma_engine_listener_set_direction(&engine, 0, 0.0f, 0.0f, -1.0f);
    return 0;
}

void* audio_load_sound(const char* path, int loop) {
    ma_sound* sound = (ma_sound*)malloc(sizeof(ma_sound));
    ma_uint32 flags = MA_SOUND_FLAG_STREAM;
    if (loop) flags |= MA_SOUND_FLAG_LOOPING;

    ma_result result = ma_sound_init_from_file(&engine, path, flags, NULL, NULL, sound);
    if (result != MA_SUCCESS) {
        free(sound);
        return NULL;
    }
    return sound;
}

void audio_set_position(void* sound_ptr, float x, float y, float z) {
    if (sound_ptr == 0) return;
    ma_sound* sound = (ma_sound*)sound_ptr;
    ma_sound_set_position(sound, x, y, z);
}

void audio_play(void* sound_ptr) {
    if (sound_ptr == 0) return;
    ma_sound* sound = (ma_sound*)sound_ptr;
    ma_sound_start(sound);
}

void audio_stop(void* sound_ptr) {
    if (sound_ptr == 0) return;
    ma_sound* sound = (ma_sound*)sound_ptr;
    ma_sound_stop(sound);
}

void audio_set_volume(void* sound_ptr, float volume) {
    if (sound_ptr == 0) return;
    ma_sound* sound = (ma_sound*)sound_ptr;
    ma_sound_set_volume(sound, volume);
}

int audio_is_playing(void* sound_ptr) {
    if (sound_ptr == 0) return 0;
    ma_sound* sound = (ma_sound*)sound_ptr;
    return ma_sound_is_playing(sound) ? 1 : 0;
}

void audio_seek_to_start(void* sound_ptr) {
    if (sound_ptr == 0) return;
    ma_sound* sound = (ma_sound*)sound_ptr;
    ma_sound_seek_to_pcm_frame(sound, 0);
}

void audio_delete_sound(void* sound_ptr) {
    if (sound_ptr == 0) return;
    ma_sound* sound = (ma_sound*)sound_ptr;
    ma_sound_uninit(sound);
    free(sound);
}

void audio_set_listener_position(float x, float y, float z) {
    ma_engine_listener_set_position(&engine, 0, x, y, z);
}

void audio_set_listener_direction(float x, float y, float z) {
    ma_engine_listener_set_direction(&engine, 0, x, y, z);
}

void audio_cleanup() {
    ma_engine_uninit(&engine);
}

void audio_set_master_volume(float volume) {
    ma_engine_set_volume(&engine, volume);
}
