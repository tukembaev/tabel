from django.contrib import admin
from . import models

@admin.register(models.Employee_Enter_Auth)
class Employee_Enter_Auth(admin.ModelAdmin):
    raw_id_fields = ('employee',)
    list_display = ('employee', 'auditorium', 'working_time')