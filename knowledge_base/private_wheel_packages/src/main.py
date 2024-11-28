# By successfully importing the previously downloaded wheel, we demonstrate that it is possible to
# include an arbitrary wheel file in your bundle directory, deploy it, and use it from within your code.
import cowsay

if __name__ == '__main__':
    cowsay.cow('Hello, world!')
