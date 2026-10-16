

from django.db import models
from django.utils.translation import gettext_lazy as _
from apps_logic.employees.models import Employee
from django.utils.timezone import now
from django.core.validators import FileExtensionValidator

class Employee_Enter_Auth(models.Model):
    employee = models.ForeignKey(Employee,  verbose_name="Сотрудник", on_delete=models.CASCADE, null=True, related_name="tabel_employees")
    auditorium = models.CharField(max_length=255)
    time = models.DateTimeField(default=now)
    date_day = models.DateField(auto_now_add=True)
    auditorium_end = models.CharField(max_length=255, blank=True, null=True)
    time_end = models.DateTimeField(blank=True, null=True)
    geo = models.CharField(max_length=6000, blank=True, null=True)
    geo_end =  models.CharField(max_length=6000, blank=True, null=True)
    working_time = models.CharField(max_length=255 , blank=True, null=True)
    similarity_index = models.CharField(max_length=255 , blank=True, null=True)
    image = models.FileField(blank=True, null=True, upload_to="media/images/")
    image_end = models.FileField(blank=True, null=True, upload_to="media/images/")
    status_info = models.CharField(blank=True, null=True,max_length=255)
    
    def __str__(self):
        return self.employee.surname + "   " + self.employee.first_name + " " + str(self.time)
    
    class Meta:
        verbose_name = "Отметки"
        verbose_name_plural = "Отметки"
        ordering = ['-date_day',]


# class FileTabel(models.Model):
#     tabel = models.ForeignKey(Employee_Enter_Auth, on_delete=models.CASCADE, related_name="files_tabels")
#     file = models.FileField(verbose_name="Документ", upload_to="media/tabels/", blank=True, null=True, validators=[FileExtensionValidator(['pdf', 'docx', 'doc', 'png', 'pptx', 'xls', 'xlsx', 'zip', 'rar', 'png', 'jpeg', 'jpg'])])

#     def __str__(self):
#         return f"{self.news} ---- {self.file}"

#     class Meta:
#         verbose_name = "Файлы новостей"
#         verbose_name_plural = "A| Файлы новостей"