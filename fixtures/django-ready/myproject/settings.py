import os
import dj_database_url

SECRET_KEY = os.environ.get('SECRET_KEY', 'default')
DEBUG = os.environ.get('RENDER') is None
ALLOWED_HOSTS = [os.environ.get('RENDER_EXTERNAL_HOSTNAME', '')]

MIDDLEWARE = [
    'django.middleware.security.SecurityMiddleware',
    'whitenoise.middleware.WhiteNoiseMiddleware',
]

DATABASES = {
    'default': dj_database_url.config(default='postgres://user:pass@localhost:5432/db')
}

BASE_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
STATIC_URL = '/static/'
STATIC_ROOT = os.path.join(BASE_DIR, 'staticfiles')
