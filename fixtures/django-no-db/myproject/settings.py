# This project has no DB-requiring apps — no auth, admin, or sessions.
import os
INSTALLED_APPS = [
    'django.contrib.staticfiles',
    'myapp',
]
MIDDLEWARE = [
    'django.middleware.security.SecurityMiddleware',
    'whitenoise.middleware.WhiteNoiseMiddleware',
]
DATABASES = {}
SECRET_KEY = os.environ.get('SECRET_KEY', 'fallback')
DEBUG = os.environ.get('RENDER') is None
ALLOWED_HOSTS = [os.environ.get('RENDER_EXTERNAL_HOSTNAME', '')]

