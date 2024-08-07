class A:
    def func():
        pass

class B:
    def func():
        pass

class C (A, B):
    def func():
        super().func()

a = A()
b = B()
c = C()

a.func()
b.func()
c.func()
